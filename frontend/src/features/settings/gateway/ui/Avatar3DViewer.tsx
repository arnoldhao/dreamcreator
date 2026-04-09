import * as React from "react";
import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { DRACOLoader } from "three/examples/jsm/loaders/DRACOLoader.js";
import { KTX2Loader } from "three/examples/jsm/loaders/KTX2Loader.js";
import { MeshoptDecoder } from "three/examples/jsm/libs/meshopt_decoder.module.js";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { VRMLoaderPlugin, VRMUtils, type VRM } from "@pixiv/three-vrm";
import {
  VRMAnimationLoaderPlugin,
  VRMLookAtQuaternionProxy,
  createVRMAnimationClip,
  type VRMAnimation,
} from "@pixiv/three-vrm-animation";

import { cn } from "@/lib/utils";
import { Skeleton } from "@/shared/ui/skeleton";

const DRACO_DECODER_PATH = "/three/draco/";
const KTX2_TRANSCODER_PATH = "/three/basis/";
const CONTROL_SETTLE_FRAME_COUNT = 18;

interface Avatar3DViewerProps {
  src: string;
  motionSrc?: string;
  className?: string;
  canvasClassName?: string;
}

type LoadState = "idle" | "loading" | "ready" | "error";

export function Avatar3DViewer({ src, motionSrc, className, canvasClassName }: Avatar3DViewerProps) {
  const containerRef = React.useRef<HTMLDivElement | null>(null);
  const rendererRef = React.useRef<THREE.WebGLRenderer | null>(null);
  const sceneRef = React.useRef<THREE.Scene | null>(null);
  const cameraRef = React.useRef<THREE.PerspectiveCamera | null>(null);
  const controlsRef = React.useRef<OrbitControls | null>(null);
  const clockRef = React.useRef(new THREE.Clock());
  const frameRef = React.useRef<number | null>(null);
  const currentObjectRef = React.useRef<THREE.Object3D | null>(null);
  const currentVrmRef = React.useRef<VRM | null>(null);
  const mixerRef = React.useRef<THREE.AnimationMixer | null>(null);
  const resizeObserverRef = React.useRef<ResizeObserver | null>(null);
  const intersectionObserverRef = React.useRef<IntersectionObserver | null>(null);
  const motionTokenRef = React.useRef(0);
  const renderLoopActiveRef = React.useRef(false);
  const interactionActiveRef = React.useRef(false);
  const settleFramesRef = React.useRef(0);
  const pageVisibleRef = React.useRef(typeof document === "undefined" ? true : document.visibilityState === "visible");
  const viewportVisibleRef = React.useRef(true);
  const [state, setState] = React.useState<LoadState>("idle");

  const canRender = React.useCallback(() => {
    return Boolean(rendererRef.current && sceneRef.current && cameraRef.current) && pageVisibleRef.current && viewportVisibleRef.current;
  }, []);

  const renderScene = React.useCallback((delta = 0) => {
    const renderer = rendererRef.current;
    const scene = sceneRef.current;
    const camera = cameraRef.current;
    if (!renderer || !scene || !camera) {
      return;
    }
    if (delta > 0 && currentVrmRef.current) {
      currentVrmRef.current.update(delta);
    }
    if (delta > 0 && mixerRef.current) {
      mixerRef.current.update(delta);
    }
    controlsRef.current?.update();
    renderer.render(scene, camera);
  }, []);

  const stopRenderLoop = React.useCallback(() => {
    if (frameRef.current !== null) {
      cancelAnimationFrame(frameRef.current);
      frameRef.current = null;
    }
    renderLoopActiveRef.current = false;
    if (clockRef.current.running) {
      clockRef.current.stop();
    }
  }, []);

  const renderLoopRef = React.useRef<() => void>(() => {});

  const ensureRenderLoop = React.useCallback(() => {
    if (renderLoopActiveRef.current || !canRender()) {
      return;
    }
    renderLoopActiveRef.current = true;
    if (!clockRef.current.running) {
      clockRef.current.start();
    }
    frameRef.current = requestAnimationFrame(() => {
      renderLoopRef.current();
    });
  }, [canRender]);

  const requestRender = React.useCallback(
    (settleFrames = 0) => {
      if (settleFrames > 0) {
        settleFramesRef.current = Math.max(settleFramesRef.current, settleFrames);
      }
      if (!canRender()) {
        return;
      }
      if (renderLoopActiveRef.current) {
        return;
      }
      if (mixerRef.current || interactionActiveRef.current || settleFramesRef.current > 0) {
        ensureRenderLoop();
        return;
      }
      renderScene(0);
    },
    [canRender, ensureRenderLoop, renderScene],
  );

  renderLoopRef.current = () => {
    if (!canRender()) {
      stopRenderLoop();
      return;
    }
    const settleFrames = settleFramesRef.current;
    const delta = clockRef.current.getDelta();
    renderScene(delta);
    if (!interactionActiveRef.current && settleFrames > 0) {
      settleFramesRef.current = settleFrames - 1;
    }
    if (mixerRef.current || interactionActiveRef.current || settleFramesRef.current > 0) {
      frameRef.current = requestAnimationFrame(() => {
        renderLoopRef.current();
      });
      renderLoopActiveRef.current = true;
      return;
    }
    stopRenderLoop();
  };

  React.useEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return;
    }

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.setSize(container.clientWidth || 1, container.clientHeight || 1);
    renderer.setClearColor(0x000000, 0);
    if ("outputColorSpace" in renderer) {
      (renderer as THREE.WebGLRenderer & { outputColorSpace: THREE.ColorSpace }).outputColorSpace =
        THREE.SRGBColorSpace;
    }

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(32, 1, 0.1, 100);
    camera.position.set(0, 1.2, 3.6);
    camera.lookAt(0, 1.1, 0);

    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.enableZoom = false;
    controls.enablePan = false;
    controlsRef.current = controls;

    const handleControlsStart = () => {
      interactionActiveRef.current = true;
      ensureRenderLoop();
    };
    const handleControlsChange = () => {
      requestRender(interactionActiveRef.current ? 0 : 1);
    };
    const handleControlsEnd = () => {
      interactionActiveRef.current = false;
      requestRender(controls.enableDamping ? CONTROL_SETTLE_FRAME_COUNT : 1);
    };
    controls.addEventListener("start", handleControlsStart);
    controls.addEventListener("change", handleControlsChange);
    controls.addEventListener("end", handleControlsEnd);

    const hemiLight = new THREE.HemisphereLight(0xffffff, 0x444444, 0.8);
    hemiLight.position.set(0, 2, 0);
    scene.add(hemiLight);
    const dirLight = new THREE.DirectionalLight(0xffffff, 1.1);
    dirLight.position.set(1, 2, 1);
    scene.add(dirLight);

    container.appendChild(renderer.domElement);
    rendererRef.current = renderer;
    sceneRef.current = scene;
    cameraRef.current = camera;
    requestRender();

    const resizeObserver = new ResizeObserver(() => {
      const nextWidth = container.clientWidth || 1;
      const nextHeight = container.clientHeight || 1;
      renderer.setSize(nextWidth, nextHeight);
      camera.aspect = nextWidth / nextHeight;
      camera.updateProjectionMatrix();
      requestRender();
    });
    resizeObserver.observe(container);
    resizeObserverRef.current = resizeObserver;

    const handleVisibilityChange = () => {
      pageVisibleRef.current = document.visibilityState === "visible";
      if (!pageVisibleRef.current) {
        stopRenderLoop();
        return;
      }
      requestRender();
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);

    if ("IntersectionObserver" in window) {
      const intersectionObserver = new IntersectionObserver(
        ([entry]) => {
          viewportVisibleRef.current = entry?.isIntersecting ?? true;
          if (!viewportVisibleRef.current) {
            stopRenderLoop();
            return;
          }
          requestRender();
        },
        { threshold: 0.01 },
      );
      intersectionObserver.observe(container);
      intersectionObserverRef.current = intersectionObserver;
    }

    return () => {
      stopRenderLoop();
      controls.removeEventListener("start", handleControlsStart);
      controls.removeEventListener("change", handleControlsChange);
      controls.removeEventListener("end", handleControlsEnd);
      if (currentObjectRef.current) {
        scene.remove(currentObjectRef.current);
        disposeObject(currentObjectRef.current);
      }
      disposeMixer(mixerRef);
      if (controlsRef.current) {
        controlsRef.current.dispose();
        controlsRef.current = null;
      }
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      resizeObserver.disconnect();
      resizeObserverRef.current = null;
      intersectionObserverRef.current?.disconnect();
      intersectionObserverRef.current = null;
      renderer.dispose();
      renderer.forceContextLoss();
      renderer.domElement.remove();
      rendererRef.current = null;
      sceneRef.current = null;
      cameraRef.current = null;
      currentObjectRef.current = null;
      currentVrmRef.current = null;
    };
  }, []);

  React.useEffect(() => {
    const renderer = rendererRef.current;
    const scene = sceneRef.current;
    const camera = cameraRef.current;
    if (!renderer || !scene || !camera) {
      return;
    }
    if (!src) {
      setState("idle");
      return;
    }

    let cancelled = false;
    const loader = new GLTFLoader();
    loader.crossOrigin = "anonymous";
    const dracoLoader = new DRACOLoader();
    dracoLoader.setDecoderPath(DRACO_DECODER_PATH);
    loader.setDRACOLoader(dracoLoader);

    const ktx2Loader = new KTX2Loader().setTranscoderPath(KTX2_TRANSCODER_PATH).detectSupport(renderer);
    loader.setKTX2Loader(ktx2Loader);
    loader.setMeshoptDecoder(MeshoptDecoder);
    loader.register((parser) => new VRMLoaderPlugin(parser));
    loader.register((parser) => new VRMAnimationLoaderPlugin(parser));

    setState("loading");
    if (currentObjectRef.current) {
      scene.remove(currentObjectRef.current);
      disposeObject(currentObjectRef.current);
      currentObjectRef.current = null;
      currentVrmRef.current = null;
    }
    disposeMixer(mixerRef);

    loader.load(
      src,
      (gltf) => {
        if (cancelled) {
          return;
        }
        if (currentObjectRef.current) {
          scene.remove(currentObjectRef.current);
          disposeObject(currentObjectRef.current);
          currentObjectRef.current = null;
          currentVrmRef.current = null;
        }

        const vrm = gltf.userData.vrm as VRM | undefined;
        if (vrm) {
          VRMUtils.rotateVRM0(vrm);
          VRMUtils.removeUnnecessaryVertices(vrm.scene);
          VRMUtils.combineSkeletons(vrm.scene);
          vrm.scene.traverse((obj) => {
            obj.frustumCulled = false;
          });
          if (vrm.lookAt && !vrm.scene.children.find((obj) => obj instanceof VRMLookAtQuaternionProxy)) {
            const proxy = new VRMLookAtQuaternionProxy(vrm.lookAt) as THREE.Object3D;
            proxy.name = "VRMLookAtQuaternionProxy";
            vrm.scene.add(proxy);
          }
          currentVrmRef.current = vrm;
          currentObjectRef.current = vrm.scene;
        } else {
          currentObjectRef.current = gltf.scene;
        }

        if (currentObjectRef.current) {
          scene.add(currentObjectRef.current);
          fitCameraToObject(camera, currentObjectRef.current, controlsRef.current);
        }

        if (motionSrc && currentVrmRef.current) {
          loadMotion(loader, motionSrc, currentVrmRef.current, mixerRef, motionTokenRef, requestRender);
        }

        setState("ready");
        requestRender();
      },
      undefined,
      () => {
        if (cancelled) {
          return;
        }
        setState("error");
        requestRender();
      }
    );

    return () => {
      cancelled = true;
      dracoLoader.dispose();
      ktx2Loader.dispose();
    };
  }, [src]);

  React.useEffect(() => {
    if (!motionSrc || !currentVrmRef.current) {
      disposeMixer(mixerRef);
      requestRender();
      return;
    }
    const loader = new GLTFLoader();
    loader.crossOrigin = "anonymous";
    loader.register((parser) => new VRMLoaderPlugin(parser));
    loader.register((parser) => new VRMAnimationLoaderPlugin(parser));
    loadMotion(loader, motionSrc, currentVrmRef.current, mixerRef, motionTokenRef, requestRender);
  }, [motionSrc, requestRender]);

  return (
    <div className={cn("relative h-full w-full", className)}>
      <div ref={containerRef} className={cn("h-full w-full", canvasClassName)} />
      {state === "loading" ? (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <div className="flex flex-col items-center gap-2">
            <Skeleton className="h-14 w-14 rounded-full bg-muted/70" />
            <Skeleton className="h-3 w-20 rounded-full bg-muted/60" />
          </div>
        </div>
      ) : null}
      {state === "error" ? (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center text-xs text-destructive">
          Failed to load
        </div>
      ) : null}
    </div>
  );
}

function fitCameraToObject(
  camera: THREE.PerspectiveCamera,
  object: THREE.Object3D,
  controls: OrbitControls | null
) {
  const box = new THREE.Box3().setFromObject(object);
  const size = box.getSize(new THREE.Vector3());
  const center = box.getCenter(new THREE.Vector3());
  const boxSize = size.length();
  if (!Number.isFinite(boxSize) || boxSize === 0) {
    return;
  }

  const halfSizeToFitOnScreen = boxSize * 0.5;
  const halfFovY = THREE.MathUtils.degToRad(camera.fov * 0.5);
  const distance = halfSizeToFitOnScreen / Math.tan(halfFovY);
  const direction = new THREE.Vector3().subVectors(camera.position, center).normalize();
  camera.position.copy(direction.multiplyScalar(distance).add(center));
  camera.near = boxSize / 100;
  camera.far = boxSize * 100;
  camera.updateProjectionMatrix();
  camera.lookAt(center);
  if (controls) {
    controls.target.copy(center);
    controls.update();
  }
}

function disposeObject(object: THREE.Object3D) {
  object.traverse((child) => {
    if (child instanceof THREE.Mesh) {
      child.geometry?.dispose?.();
      const material = child.material as THREE.Material | THREE.Material[];
      if (Array.isArray(material)) {
        material.forEach((item) => item.dispose());
      } else {
        material?.dispose?.();
      }
    }
  });
}

function disposeMixer(mixerRef: React.MutableRefObject<THREE.AnimationMixer | null>) {
  if (mixerRef.current) {
    mixerRef.current.stopAllAction();
    mixerRef.current = null;
  }
}

function loadMotion(
  loader: GLTFLoader,
  motionSrc: string,
  vrm: VRM,
  mixerRef: React.MutableRefObject<THREE.AnimationMixer | null>,
  tokenRef: React.MutableRefObject<number>,
  requestRender: (settleFrames?: number) => void
) {
  const token = tokenRef.current + 1;
  tokenRef.current = token;

  loader.load(
    motionSrc,
    (gltf) => {
      if (tokenRef.current !== token) {
        return;
      }
      const animations = gltf.userData.vrmAnimations as VRMAnimation[] | undefined;
      const vrmAnimation = animations?.[0];
      if (!vrmAnimation) {
        disposeMixer(mixerRef);
        requestRender();
        return;
      }
      if (!mixerRef.current) {
        mixerRef.current = new THREE.AnimationMixer(vrm.scene);
      }
      const mixer = mixerRef.current;
      mixer.stopAllAction();
      const clip = createVRMAnimationClip(vrmAnimation, vrm);
      mixer.clipAction(clip).play();
      requestRender(1);
    },
    undefined,
    () => {
      if (tokenRef.current === token) {
        disposeMixer(mixerRef);
        requestRender();
      }
    }
  );
}
