import type { A2UIComponent, A2UISurface } from '../types'

type Props = {
  surface?: A2UISurface
}

export function A2UIRenderer({ surface }: Props) {
  if (!surface) {
    return (
      <section className="surface empty">
        <h2>A2UI Surface</h2>
        <p>Agent 暂未返回声明式 UI。</p>
      </section>
    )
  }

  return (
    <section className="surface">
      <h2>{surface.title ?? surface.id}</h2>
      <div className="componentStack">
        {surface.components.map((component) => (
          <RenderComponent key={component.id} component={component} />
        ))}
      </div>
    </section>
  )
}

function RenderComponent({ component }: { component: A2UIComponent }) {
  const props = component.props ?? {}

  switch (component.type) {
    case 'text':
      return <p className="a2uiText">{String(props.text ?? '')}</p>
    case 'card':
      return (
        <article className="a2uiCard">
          <h3>{String(props.title ?? 'Card')}</h3>
          <p>{String(props.body ?? '')}</p>
          {component.children?.map((child) => (
            <RenderComponent key={child.id} component={child} />
          ))}
        </article>
      )
    case 'button':
      return <button className="a2uiButton">{String(props.label ?? 'Action')}</button>
    case 'list':
      return (
        <ul className="a2uiList">
          {Array.isArray(props.items) && props.items.map((item, index) => <li key={index}>{String(item)}</li>)}
        </ul>
      )
    default:
      return (
        <pre className="unknownComponent">
          {JSON.stringify(component, null, 2)}
        </pre>
      )
  }
}
